var profiles = []

// Function to fetch phone numbers from AWS
function fetchPhoneNumbers() {
    var accessKeyId = $("#access_key_id").val()
    var secretKey = $("#secret_key").val()
    var region = $("#region").val()
    
    if (!accessKeyId || !secretKey || !region) {
        $("#sms_from").prop('disabled', true)
        $("#sms_from").html('<option value="">Enter AWS credentials to load phone numbers</option>')
        return
    }
    
    $("#phone_loading").show()
    $("#sms_from").prop('disabled', true)
    $("#sms_from").html('<option value="">Loading phone numbers...</option>')
    
    api.SMS.phoneNumbers({
        access_key_id: accessKeyId,
        secret_key: secretKey,
        region: region
    })
    .success(function(response) {
        $("#phone_loading").hide()
        if (response.success && response.phone_numbers) {
            var options = '<option value="">Select Phone Number</option>'
            response.phone_numbers.forEach(function(phoneNumber) {
                options += '<option value="' + phoneNumber + '">' + phoneNumber + '</option>'
            })
            $("#sms_from").html(options)
            $("#sms_from").prop('disabled', false)
        } else {
            $("#sms_from").html('<option value="">Error loading phone numbers</option>')
            $("#sms_from").prop('disabled', true)
            modalError(response.message || "Failed to load phone numbers")
        }
    })
    .error(function(xhr) {
        $("#phone_loading").hide()
        $("#sms_from").html('<option value="">Error loading phone numbers</option>')
        $("#sms_from").prop('disabled', true)
        var errorMsg = "Failed to load phone numbers"
        if (xhr.responseJSON && xhr.responseJSON.message) {
            errorMsg = xhr.responseJSON.message
        }
        modalError(errorMsg)
    })
}

// Save attempts to POST to /smtp/
function save(idx) {
    var profile = {}
    $.each($("#headersTable").DataTable().rows().data(), function (i, header) {
        profile.headers.push({
            key: unescapeHtml(header[0]),
            value: unescapeHtml(header[1]),
        })
    })
    profile.name = $("#name").val()
    profile.access_key_id = $("#access_key_id").val()
    profile.secret_key = $("#secret_key").val()
    profile.region = $("#region").val()
    profile.sms_from = $("#sms_from").val()

    if (idx != -1) {
        profile.id = profiles[idx].id
        api.SMSId.put(profile)
            .success(function (data) {
                successFlash("Profile edited successfully!")
                load()
                dismiss()
            })
            .error(function (data) {
                modalError(data.responseJSON.message)
            })
    } else {
        // Submit the profile
        api.SMS.post(profile)
            .success(function (data) {
                successFlash("Profile added successfully!")
                load()
                dismiss()
            })
            .error(function (data) {
                modalError(data.responseJSON.message)
            })
    }
}

function dismiss() {
    $("#modal\\.flashes").empty()
    $("#name").val("")
    $("#access_key_id").val("")
    $("#secret_key").val("")
    $("#region").val("us-east-1")
    $("#sms_from").val("")
    $("#sms_from").html('<option value="">Enter AWS credentials to load phone numbers</option>')
    $("#sms_from").prop('disabled', true)
    $("#phone_loading").hide()
    $("#headersTable").dataTable().DataTable().clear().draw()
    $("#modal").modal('hide')
}

var dismissSendTestEmailModal = function () {
    $("#sendTestEmailModal\\.flashes").empty()
    $("#sendTestModalSubmit").html("<i class='fa fa-envelope'></i> Send")
}

var deleteProfile = function (idx) {
    Swal.fire({
        title: "Are you sure?",
        text: "This will delete the sending profile. This can't be undone!",
        type: "warning",
        animation: false,
        showCancelButton: true,
        confirmButtonText: "Delete " + escapeHtml(profiles[idx].name),
        confirmButtonColor: "#428bca",
        reverseButtons: true,
        allowOutsideClick: false,
        preConfirm: function () {
            return new Promise(function (resolve, reject) {
                api.SMSId.delete(profiles[idx].id)
                    .success(function (msg) {
                        resolve()
                    })
                    .error(function (data) {
                        reject(data.responseJSON.message)
                    })
            })
        }
    }).then(function (result) {
        if (result.value){
            Swal.fire(
                'Sending Profile Deleted!',
                'This sending profile has been deleted!',
                'success'
            );
        }
        $('button:contains("OK")').on('click', function () {
            location.reload()
        })
    })
}

function edit(idx) {
    headers = $("#headersTable").dataTable({
        destroy: true, // Destroy any other instantiated table - http://datatables.net/manual/tech-notes/3#destroy
        columnDefs: [{
            orderable: false,
            targets: "no-sort"
        }]
    })

    $("#modalSubmit").unbind('click').click(function () {
        save(idx)
    })
    var profile = {}
    if (idx != -1) {
        $("#profileModalLabel").text("Edit Sending Profile")
        profile = profiles[idx]
        $("#name").val(profile.name)
        $("#access_key_id").val(profile.access_key_id)
        $("#secret_key").val(profile.secret_key)
        $("#region").val(profile.region)
        // For editing, populate the phone number dropdown with the saved value
        if (profile.sms_from) {
            $("#sms_from").html('<option value="' + profile.sms_from + '">' + profile.sms_from + '</option>')
            $("#sms_from").val(profile.sms_from)
            $("#sms_from").prop('disabled', false)
        } else {
            $("#sms_from").html('<option value="">Enter AWS credentials to load phone numbers</option>')
            $("#sms_from").prop('disabled', true)
        }
    } else {
        $("#profileModalLabel").text("New Sending Profile")
        // Reset to default state for new profile
        $("#region").val("us-east-1")
        $("#sms_from").html('<option value="">Enter AWS credentials to load phone numbers</option>')
        $("#sms_from").prop('disabled', true)
    }
}

function copy(idx) {
    $("#modalSubmit").unbind('click').click(function () {
        save(-1)
    })
    var profile = {}
    profile = profiles[idx]
    $("#name").val("Copy of " + profile.name)
    $("#access_key_id").val(profile.access_key_id)
    $("#secret_key").val(profile.secret_key)
    $("#region").val(profile.region)
    // For copying, populate the phone number dropdown with the saved value
    if (profile.sms_from) {
        $("#sms_from").html('<option value="' + profile.sms_from + '">' + profile.sms_from + '</option>')
        $("#sms_from").val(profile.sms_from)
        $("#sms_from").prop('disabled', false)
    } else {
        $("#sms_from").html('<option value="">Enter AWS credentials to load phone numbers</option>')
        $("#sms_from").prop('disabled', true)
    }
}

function load() {
    $("#profileTable").hide()
    $("#emptyMessage").hide()
    $("#loading").show()
    api.SMS.get()
        .success(function (ss) {
            profiles = ss
            $("#loading").hide()
            if (profiles.length > 0) {
                $("#profileTable").show()
                profileTable = $("#profileTable").DataTable({
                    destroy: true,
                    columnDefs: [{
                        orderable: false,
                        targets: "no-sort"
                    }]
                });
                profileTable.clear()
                profileRows = []
                $.each(profiles, function (i, profile) {
                    profileRows.push([
                        escapeHtml(profile.name),
                        profile.interface_type,
                        moment(profile.modified_date).format('MMMM Do YYYY, h:mm:ss a'),
                        "<div class='pull-right'><span data-toggle='modal' data-backdrop='static' data-target='#modal'><button class='btn btn-primary' data-toggle='tooltip' data-placement='left' title='Edit Profile' onclick='edit(" + i + ")'>\
                    <i class='fa fa-pencil'></i>\
                    </button></span>\
		    <span data-toggle='modal' data-target='#modal'><button class='btn btn-primary' data-toggle='tooltip' data-placement='left' title='Copy Profile' onclick='copy(" + i + ")'>\
                    <i class='fa fa-copy'></i>\
                    </button></span>\
                    <button class='btn btn-danger' data-toggle='tooltip' data-placement='left' title='Delete Profile' onclick='deleteProfile(" + i + ")'>\
                    <i class='fa fa-trash-o'></i>\
                    </button></div>"
                    ])
                })
                profileTable.rows.add(profileRows).draw()
                $('[data-toggle="tooltip"]').tooltip()
            } else {
                $("#emptyMessage").show()
            }
        })
        .error(function () {
            $("#loading").hide()
            errorFlash("Error fetching profiles")
        })
}

function addCustomHeader(header, value) {
    // Create new data row.
    var newRow = [
        escapeHtml(header),
        escapeHtml(value),
        '<span style="cursor:pointer;"><i class="fa fa-trash-o"></i></span>'
    ];

    // Check table to see if header already exists.
    var headersTable = headers.DataTable();
    var existingRowIndex = headersTable
        .column(0) // Email column has index of 2
        .data()
        .indexOf(escapeHtml(header));

    // Update or add new row as necessary.
    if (existingRowIndex >= 0) {
        headersTable
            .row(existingRowIndex, {
                order: "index"
            })
            .data(newRow);
    } else {
        headersTable.row.add(newRow);
    }
    headersTable.draw();
}

$(document).ready(function () {
    // Setup multiple modals
    // Code based on http://miles-by-motorcycle.com/static/bootstrap-modal/index.html
    $('.modal').on('hidden.bs.modal', function (event) {
        $(this).removeClass('fv-modal-stack');
        $('body').data('fv_open_modals', $('body').data('fv_open_modals') - 1);
    });
    $('.modal').on('shown.bs.modal', function (event) {
        // Keep track of the number of open modals
        if (typeof ($('body').data('fv_open_modals')) == 'undefined') {
            $('body').data('fv_open_modals', 0);
        }
        // if the z-index of this modal has been set, ignore.
        if ($(this).hasClass('fv-modal-stack')) {
            return;
        }
        $(this).addClass('fv-modal-stack');
        // Increment the number of open modals
        $('body').data('fv_open_modals', $('body').data('fv_open_modals') + 1);
        // Setup the appropriate z-index
        $(this).css('z-index', 1040 + (10 * $('body').data('fv_open_modals')));
        $('.modal-backdrop').not('.fv-modal-stack').css('z-index', 1039 + (10 * $('body').data('fv_open_modals')));
        $('.modal-backdrop').not('fv-modal-stack').addClass('fv-modal-stack');
    });
    $.fn.modal.Constructor.prototype.enforceFocus = function () {
        $(document)
            .off('focusin.bs.modal') // guard against infinite focus loop
            .on('focusin.bs.modal', $.proxy(function (e) {
                if (
                    this.$element[0] !== e.target && !this.$element.has(e.target).length
                    // CKEditor compatibility fix start.
                    &&
                    !$(e.target).closest('.cke_dialog, .cke').length
                    // CKEditor compatibility fix end.
                ) {
                    this.$element.trigger('focus');
                }
            }, this));
    };
    // Scrollbar fix - https://stackoverflow.com/questions/19305821/multiple-modals-overlay
    $(document).on('hidden.bs.modal', '.modal', function () {
        $('.modal:visible').length && $(document.body).addClass('modal-open');
    });
    $('#modal').on('hidden.bs.modal', function (event) {
        dismiss()
    });
    $("#sendTestEmailModal").on("hidden.bs.modal", function (event) {
        dismissSendTestEmailModal()
    })
    // Code to deal with custom email headers
    $("#addCustomHeader").on('click', function () {
        headerKey = $("#headerKey").val();
        headerValue = $("#headerValue").val();

        if (headerKey == "" || headerValue == "") {
            return false;
        }
        addCustomHeader(headerKey, headerValue);
        // Reset user input.
        $("#headerKey").val('');
        $("#headerValue").val('');
        $("#headerKey").focus();
        return false;
    });
    // Handle Deletion
    $("#headersTable").on("click", "span>i.fa-trash-o", function () {
        headers.DataTable()
            .row($(this).parents('tr'))
            .remove()
            .draw();
    });
    
    // Add event listeners for credential fields to fetch phone numbers
    $("#access_key_id, #secret_key").on('input', function() {
        // Debounce the function call to avoid too many API calls
        clearTimeout(window.phoneNumberTimeout);
        window.phoneNumberTimeout = setTimeout(function() {
            fetchPhoneNumbers();
        }, 1000); // Wait 1 second after user stops typing
    });
    
    // Also trigger on paste events
    $("#access_key_id, #secret_key").on('paste', function() {
        var self = this;
        setTimeout(function() {
            clearTimeout(window.phoneNumberTimeout);
            window.phoneNumberTimeout = setTimeout(function() {
                fetchPhoneNumbers();
            }, 1000);
        }, 100);
    });
    
    load()
})
