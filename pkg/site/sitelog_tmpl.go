package site

// The sitelog template
const sitelogTempl = `
     XXXX Site Information Form (site log)
     International GNSS Service
     See Instructions at:
       ftp://igs.org/pub/station/general/sitelog_instr.txt

0.   Form

     Prepared by (full name)  : {{.FormInfo.PreparedBy}}
     Date Prepared            : {{.FormInfo.DatePrepared | printDate}}
     Report Type              : (NEW/UPDATE)
     If Update:
      Previous Site Log       : (ssss_ccyymmdd.log)
      Modified/Added Sections : (n.n,n.n,...)


1.   Site Identification of the GNSS Monument
{{with .Ident}}
     Site Name                : {{.Name}}
     Four Character ID        : {{.FourCharacterID}}
     Monument Inscription     : {{.MonumentInscription}}
     IERS DOMES Number        : {{.DOMESNumber}}
     CDP Number               : {{.CDPNumber}}
     Monument Description     : {{.MonumentDescription}}
       Height of the Monument : {{.HeightOfMonument}}
       Monument Foundation    : {{.MonumentFoundation}}
       Foundation Depth       : {{.FoundationDepth}}
     Marker Description       : {{.MarkerDescription}}
     Date Installed           : {{.DateInstalled | printDateTime}}
     Geologic Characteristic  : {{.GeologicCharacteristic}}
       Bedrock Type           : {{.BedrockType}}
       Bedrock Condition      : {{.BedrockCondition}}
       Fracture Spacing       : {{.FractureSpacing}}
       Fault zones nearby     : {{.FaultZonesNearby}}
         Distance/activity    : {{.DistanceActivity}}
     Additional Information   : (multiple lines)
{{end}}

2.   Site Location Information
{{with .Location}}
     City or Town             : {{.City}}
     State or Province        : {{.State}}
     Country                  : {{.Country}}
     Tectonic Plate           : {{.TectonicPlate}}
     Approximate Position (ITRF)
       X coordinate (m)       : {{index .ApproximatePosition.CartesianPosition.Coordinates 0 | printf "%.03f"}} 
       Y coordinate (m)       : {{index .ApproximatePosition.CartesianPosition.Coordinates 1 | printf "%.03f"}} 
       Z coordinate (m)       : {{index .ApproximatePosition.CartesianPosition.Coordinates 2 | printf "%.03f"}} 
       Latitude (N is +)      : (+/-DDMMSS.SS)
       Longitude (E is +)     : (+/-DDDMMSS.SS)
       Elevation (m,ellips.)  : (F7.1)
     Additional Information   : (multiple lines)
{{end}}

3.   GNSS Receiver Information
{{range $i, $recv := .Receivers}}
3.{{$i | add 1}}  Receiver Type            : {{$recv.Type}}
     Satellite System         : {{$recv.SatSystems | html}}
     Serial Number            : {{$recv.SerialNum}}
     Firmware Version         : {{$recv.Firmware}}
     Elevation Cutoff Setting : {{$recv.ElevationCutoff}}
     Date Installed           : {{$recv.DateInstalled | printDateTime}}
     Date Removed             : {{$recv.DateRemoved | printDateTime}}
     Temperature Stabiliz.    : {{$recv.TemperatureStabiliz | html}}
     Additional Information   : (multiple lines)
{{end}}
3.x  Receiver Type            : (A20, from rcvr_ant.tab; see instructions)
     Satellite System         : (GPS+GLO+GAL+BDS+QZSS+SBAS)
     Serial Number            : (A20, but note the first A5 is used in SINEX)
     Firmware Version         : (A11)
     Elevation Cutoff Setting : (deg)
     Date Installed           : (CCYY-MM-DDThh:mmZ)
     Date Removed             : (CCYY-MM-DDThh:mmZ)
     Temperature Stabiliz.    : (none or tolerance in degrees C)
     Additional Information   : (multiple lines)


4.   GNSS Antenna Information
{{range $i, $ant := .Antennas}}
4.{{$i | add 1}}  Antenna Type             : {{$ant.Type}}
     Serial Number            : {{$ant.SerialNum}}
     Antenna Reference Point  : {{$ant.ReferencePoint}}
     Marker->ARP Up Ecc. (m)  : {{printf "%.04f" $ant.EccUp}}
     Marker->ARP North Ecc(m) : {{printf "%.04f" $ant.EccNorth}}
     Marker->ARP East Ecc(m)  : {{printf "%.04f" $ant.EccEast}}
     Alignment from True N    : {{$ant.AlignmentFromTrueNorth}} 
     Antenna Radome Type      : {{$ant.Radome}}
     Radome Serial Number     : {{$ant.RadomeSerialNum}}
     Antenna Cable Type       : {{$ant.CableType}} 
     Antenna Cable Length     : {{$ant.CableLength}}
     Date Installed           : {{$ant.DateInstalled | printDateTime}}
     Date Removed             : {{$ant.DateRemoved | printDateTime}}
     Additional Information   : (multiple lines)
{{end}}
4.x  Antenna Type             : (A20, from rcvr_ant.tab; see instructions)
     Serial Number            : (A*, but note the first A5 is used in SINEX)
     Antenna Reference Point  : (BPA/BCR/XXX from "antenna.gra"; see instr.)
     Marker->ARP Up Ecc. (m)  : (F8.4)
     Marker->ARP North Ecc(m) : (F8.4)
     Marker->ARP East Ecc(m)  : (F8.4)
     Alignment from True N    : (deg; + is clockwise/east)
     Antenna Radome Type      : (A4 from rcvr_ant.tab; see instructions)
     Radome Serial Number     : 
     Antenna Cable Type       : (vendor & type number)
     Antenna Cable Length     : (m)
     Date Installed           : (CCYY-MM-DDThh:mmZ)
     Date Removed             : (CCYY-MM-DDThh:mmZ)
     Additional Information   : (multiple lines)

5.   Surveyed Local Ties

5.x  Tied Marker Name         : 
     Tied Marker Usage        : (SLR/VLBI/LOCAL CONTROL/FOOTPRINT/etc)
     Tied Marker CDP Number   : (A4)
     Tied Marker DOMES Number : (A9)
     Differential Components from GNSS Marker to the tied monument (ITRS)
       dx (m)                 : (m)
       dy (m)                 : (m)
       dz (m)                 : (m)
     Accuracy (mm)            : (mm)
     Survey method            : (GPS CAMPAIGN/TRILATERATION/TRIANGULATION/etc)
     Date Measured            : (CCYY-MM-DDThh:mmZ)
     Additional Information   : (multiple lines)


6.   Frequency Standard

6.1  Standard Type            : (INTERNAL or EXTERNAL H-MASER/CESIUM/etc)
       Input Frequency        : (if external)
       Effective Dates        : (CCYY-MM-DD/CCYY-MM-DD)
       Notes                  : (multiple lines)

6.x  Standard Type            : (INTERNAL or EXTERNAL H-MASER/CESIUM/etc)
       Input Frequency        : (if external)
       Effective Dates        : (CCYY-MM-DD/CCYY-MM-DD)
       Notes                  : (multiple lines)


7.   Collocation Information

7.1  Instrumentation Type     : (GPS/GLONASS/DORIS/PRARE/SLR/VLBI/TIME/etc)
       Status                 : (PERMANENT/MOBILE)
       Effective Dates        : (CCYY-MM-DD/CCYY-MM-DD)
       Notes                  : (multiple lines)

7.x  Instrumentation Type     : (GPS/GLONASS/DORIS/PRARE/SLR/VLBI/TIME/etc)
       Status                 : (PERMANENT/MOBILE)
       Effective Dates        : (CCYY-MM-DD/CCYY-MM-DD)
       Notes                  : (multiple lines)


8.   Meteorological Instrumentation

8.1.1 Humidity Sensor Model   : 
       Manufacturer           : 
       Serial Number          : 
       Data Sampling Interval : (sec)
       Accuracy (% rel h)     : (% rel h)
       Aspiration             : (UNASPIRATED/NATURAL/FAN/etc)
       Height Diff to Ant     : (m)
       Calibration date       : (CCYY-MM-DD)
       Effective Dates        : (CCYY-MM-DD/CCYY-MM-DD)
       Notes                  : (multiple lines)

8.1.x Humidity Sensor Model   : 
       Manufacturer           : 
       Serial Number          : 
       Data Sampling Interval : (sec)
       Accuracy (% rel h)     : (% rel h)
       Aspiration             : (UNASPIRATED/NATURAL/FAN/etc)
       Height Diff to Ant     : (m)
       Calibration date       : (CCYY-MM-DD)
       Effective Dates        : (CCYY-MM-DD/CCYY-MM-DD)
       Notes                  : (multiple lines)

8.2.1 Pressure Sensor Model   : 
       Manufacturer           : 
       Serial Number          : 
       Data Sampling Interval : (sec)
       Accuracy               : (hPa)
       Height Diff to Ant     : (m)
       Calibration date       : (CCYY-MM-DD)
       Effective Dates        : (CCYY-MM-DD/CCYY-MM-DD)
       Notes                  : (multiple lines)

8.2.x Pressure Sensor Model   : 
       Manufacturer           : 
       Serial Number          : 
       Data Sampling Interval : (sec)
       Accuracy               : (hPa)
       Height Diff to Ant     : (m)
       Calibration date       : (CCYY-MM-DD)
       Effective Dates        : (CCYY-MM-DD/CCYY-MM-DD)
       Notes                  : (multiple lines)

8.3.1 Temp. Sensor Model      : 
       Manufacturer           : 
       Serial Number          : 
       Data Sampling Interval : (sec)
       Accuracy               : (deg C)
       Aspiration             : (UNASPIRATED/NATURAL/FAN/etc)
       Height Diff to Ant     : (m)
       Calibration date       : (CCYY-MM-DD)
       Effective Dates        : (CCYY-MM-DD/CCYY-MM-DD)
       Notes                  : (multiple lines)

8.3.x Temp. Sensor Model      : 
       Manufacturer           : 
       Serial Number          : 
       Data Sampling Interval : (sec)
       Accuracy               : (deg C)
       Aspiration             : (UNASPIRATED/NATURAL/FAN/etc)
       Height Diff to Ant     : (m)
       Calibration date       : (CCYY-MM-DD)
       Effective Dates        : (CCYY-MM-DD/CCYY-MM-DD)
       Notes                  : (multiple lines)

8.4.1 Water Vapor Radiometer  : 
       Manufacturer           : 
       Serial Number          : 
       Distance to Antenna    : (m)
       Height Diff to Ant     : (m)
       Calibration date       : (CCYY-MM-DD)
       Effective Dates        : (CCYY-MM-DD/CCYY-MM-DD)
       Notes                  : (multiple lines)

8.4.x Water Vapor Radiometer  : 
       Manufacturer           : 
       Serial Number          : 
       Distance to Antenna    : (m)
       Height Diff to Ant     : (m)
       Calibration date       : (CCYY-MM-DD)
       Effective Dates        : (CCYY-MM-DD/CCYY-MM-DD)
       Notes                  : (multiple lines)

8.5.1 Other Instrumentation   : (multiple lines)

8.5.x Other Instrumentation   : (multiple lines)


9.  Local Ongoing Conditions Possibly Affecting Computed Position

9.1.1 Radio Interferences     : (TV/CELL PHONE ANTENNA/RADAR/etc)
       Observed Degradations  : (SN RATIO/DATA GAPS/etc)
       Effective Dates        : (CCYY-MM-DD/CCYY-MM-DD)
       Additional Information : (multiple lines)

9.1.x Radio Interferences     : (TV/CELL PHONE ANTENNA/RADAR/etc)
       Observed Degradations  : (SN RATIO/DATA GAPS/etc)
       Effective Dates        : (CCYY-MM-DD/CCYY-MM-DD)
       Additional Information : (multiple lines)

9.2.1 Multipath Sources       : (METAL ROOF/DOME/VLBI ANTENNA/etc)
       Effective Dates        : (CCYY-MM-DD/CCYY-MM-DD)
       Additional Information : (multiple lines)

9.2.x Multipath Sources       : (METAL ROOF/DOME/VLBI ANTENNA/etc)
       Effective Dates        : (CCYY-MM-DD/CCYY-MM-DD)
       Additional Information : (multiple lines)

9.3.1 Signal Obstructions     : (TREES/BUILDINGS/etc)
       Effective Dates        : (CCYY-MM-DD/CCYY-MM-DD)
       Additional Information : (multiple lines)

9.3.x Signal Obstructions     : (TREES/BUILDINGS/etc)
       Effective Dates        : (CCYY-MM-DD/CCYY-MM-DD)
       Additional Information : (multiple lines)

10.  Local Episodic Effects Possibly Affecting Data Quality

10.1 Date                     : (CCYY-MM-DD/CCYY-MM-DD)
     Event                    : (TREE CLEARING/CONSTRUCTION/etc)

10.x Date                     : (CCYY-MM-DD/CCYY-MM-DD)
     Event                    : (TREE CLEARING/CONSTRUCTION/etc)

11.   On-Site, Point of Contact Agency Information
{{with $priContact := index .Contacts 0}}
     Agency                   : {{$priContact.Party.OrganisationName}}
     Preferred Abbreviation   : {{$priContact.Party.Abbreviation}}
     Mailing Address          : (multiple lines)
     Primary Contact
       Contact Name           : {{$priContact.Party.IndividualName}}
       Telephone (primary)    :
       Telephone (secondary)  : 
       Fax                    : 
       E-mail                 : {{with $priContact.Party.ContactInfo.Address.EmailAddresses}}{{index . 0}} {{end}}{{end}}
     Secondary Contact
       Contact Name           : 
       Telephone (primary)    : 
       Telephone (secondary)  : 
       Fax                    : 
       E-mail                 : 
     Additional Information   : (multiple lines)

12.  Responsible Agency (if different from 11.)

     Agency                   : (multiple lines)
     Preferred Abbreviation   : (A10)
     Mailing Address          : (multiple lines)
     Primary Contact
       Contact Name           : 
       Telephone (primary)    : 
       Telephone (secondary)  : 
       Fax                    : 
       E-mail                 : 
     Secondary Contact
       Contact Name           : 
       Telephone (primary)    : 
       Telephone (secondary)  : 
       Fax                    : 
       E-mail                 : 
     Additional Information   : (multiple lines)


13.  More Information

     Primary Data Center      :
     Secondary Data Center    :
     URL for More Information : 
     Hardcopy on File
       Site Map               : (Y or URL)
       Site Diagram           : (Y or URL)
       Horizon Mask           : (Y or URL)
       Monument Description   : (Y or URL)
       Site Pictures          : (Y or URL)
     Additional Information   : (multiple lines)
     Antenna Graphics with Dimensions

     (insert text graphic from file antenna.gra)
`
